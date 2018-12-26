
dofile 'deferinit.lua'

a = 0;
b = 0;

-- f uses defer, boilerplate:  
function f(...)
   orig = {...}
   
   local __defers={}
   
  
  -- end boilerplate, begin custom:

   -- The ordering of entries in the __zeroret and __namedNames arrays
   -- must match the declaration order of the return parameters.
   local __zeroret = {0,0}
  -- even anonymous returns get names here.
  local __namedNames = {"ret0", "ret1"}

  local __actual=function()

     local __defer_func = function(a)
        -- capture any arguments at defer call point
        local a = a
        __defers[1+#__defers] = function() 
            print("first defer running, a=", a, " b=",b)
            b = b + 3
            ret0 = (ret0+1) * 3
            ret1 = ret1 + 1
            recov = recover()
            print("defer 1 recovered ", recov)
        end
     end
     __defer_func(a)
     
     local __defer_func = function()
        __defers[1+#__defers] = function()
           print("second defer running, a=", a, " b=",b)
           b = b * 7
           ret0 = ret0 + 100
           ret1 = ret1 + 100
           recov = recover()
           print("second defer, recov is ", recov)
           panic("panic-in-defer-2")

           -- sadly, a raw error will result in loss of the "in-defer-2" value
           -- because of problems in luajit with recursive handling of
           -- xpcalls (it doesn't like them). So try to explicitly panic
           -- whenever possible instead of allowing an error to occur!
           -- https://stackoverflow.com/questions/48202338/on-latest-luajit-2-1-0-beta3-is-recursive-xpcall-possible
           --
           --error("error-in-defer-2") 
        end
     end
     __defer_func()

     a = 1
     b = 1

    panic("ouch")
     
     return b, 58

  end -- end of __actual
  
  return __actuallyCall("f", __actual, __namedNames, __zeroret, __defers)
end

f1, f2 = f()
print("f1 = ",f1, " f2=", f2)

--[[
dofile 'flow3.lua'
panicHandler running with err =	flow3.lua:23: ouch	 and #defer = 	2
second defer running, a=	1	 b=	1
second defer, recov is 	flow3.lua:23: ouch
panic path defer call result: i=	1	  v=	false
panic path defer call result: i=	2	  v=	error in error handling
first defer running, a=	0	 b=	7
defer 1 recovered 	panic-in-defer-2
panic path defer call result: i=	1	  v=	true
checking for panic still un-caught...	nil
at f return point, ret0 = 	303	 ret1=	101
f1 = 	303	 f2=	101
> --]]

